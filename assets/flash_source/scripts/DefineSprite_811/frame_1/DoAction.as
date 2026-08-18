if(_X < 140)
{
   xx = 3;
}
if(_X < 80)
{
   xx = 2;
}
if(_X < 15)
{
   xx = 1;
}
if(_Y < 660)
{
   yy = 4;
}
if(_Y < 600)
{
   yy = 3;
}
if(_Y < 540)
{
   yy = 2;
}
if(_Y < 480)
{
   yy = 1;
}
if(_Y < 420)
{
   yy = 0;
}
NUMBER = yy * 3 + xx;
shirt.gotoAndStop(NUMBER);
asdf = _parent._parent;
this.onRelease = function()
{
   asdf.player.shirt.gotoAndStop(NUMBER);
   updatetime = 1;
   asdf.slideback();
   asdf.slide = 1;
};
this.onRollOver = function()
{
   asdf.displaytext.text = shirt.id;
};
