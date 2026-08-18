asdf = _parent._parent;
i = 0;
while(i < _root.activeplayers.length)
{
   if(_root.activeplayers[i].frame.hitTest(asdf._x + 70 * asdf.facing,asdf._y - 30,true) || _root.activeplayers[i].frame.hitTest(asdf._x + 35 * asdf.facing,asdf._y - 30,true))
   {
      _root.activeplayers[i].vx += 50 * asdf.facing;
      _root.activeplayers[i].hitnumber += 1;
      _root.activeplayers[i].hittimer = 0;
      qwer = _root.activeplayers[i].hitnumber;
      _root.CP("fx_combo",asdf._x + 100 * asdf.facing,asdf._y - 80,0,-7);
   }
   i++;
}
