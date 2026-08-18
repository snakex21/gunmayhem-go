stop();
this.onRollOver = function()
{
   gotoAndPlay(2);
   _parent.modeallplay();
};
this.onRollOut = function()
{
   gotoAndPlay(9);
};
this.onRelease = function()
{
   _parent.modealldisable();
   chosen._alpha = 100;
   switch(_name)
   {
      case "modebtn1":
         _parent.modedisplay.gotoAndStop(_parent.modedisplay._totalframes - 0);
         _root.gamemode = 1;
         break;
      case "modebtn2":
         _parent.modedisplay.gotoAndStop(_parent.modedisplay._totalframes - 1);
         _root.gamemode = 2;
         break;
      case "modebtn3":
         _parent.modedisplay.gotoAndStop(_parent.modedisplay._totalframes - 2);
         _root.gamemode = 3;
         break;
      case "modebtn4":
         _parent.modedisplay.gotoAndStop(_parent.modedisplay._totalframes - 3);
         _root.gamemode = 4;
         break;
      case "modebtn5":
         _parent.modedisplay.gotoAndStop(_parent.modedisplay._totalframes - 4);
         _root.gamemode = 5;
         break;
      case "modebtn6":
         _parent.modedisplay.gotoAndStop(_parent.modedisplay._totalframes - 5);
         _root.gamemode = 6;
      default:
         return;
   }
};
