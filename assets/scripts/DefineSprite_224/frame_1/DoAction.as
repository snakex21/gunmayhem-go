stop();
switch(_parent._parent._parent.PLAYERNUMBER)
{
   case 1:
      gotoAndStop(_root.p1shirt);
      break;
   case 2:
      gotoAndStop(_root.p2shirt);
      break;
   case 3:
      gotoAndStop(_root.p3shirt);
      break;
   case 4:
      gotoAndStop(_root.p4shirt);
}
if(_parent._parent._parent.nametag.nametext.text == "Homer")
{
   gotoAndStop(32);
}
